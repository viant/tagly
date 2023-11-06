# tagly

GoLang tag toolbox

[![GoReportCard](https://goreportcard.com/badge/github.com/viant/tagly)](https://goreportcard.com/report/github.com/viant/tagly)
[![GoDoc](https://godoc.org/github.com/viant/tagly?status.svg)](https://godoc.org/github.com/viant/tagly)

This library is compatible with Go 1.17+

Please refer to [`CHANGELOG.md`](CHANGELOG.md) if you encounter breaking changes.

- [Motivation](#motivation)
- [Usage](#usage)
- [Contribution](#contributing-to-tagly)
- [License](#license)

## Motivation

This project provides go struct tag utility to parse complex tags, alongside common tag functionality

   `tagNameX:"name,key1=value1,key2,repeated={1,2,3},keyN=valueN" ` 

This project also defines formatting utilities:


## Usage

#### Formatting tag

```go

type (
	Bar struct {
	    Info string	
    }   
	Dummy struct {
        ID              string      `format:"Id"`
        At              time.Time   `format:"tz=UTC,dateFormat:yyyy-MM-dd hh:mm`
        Internal bool               `format:"-"`
		Bar                         `format:",inline"`
		AttrX string                `format:",caseFormat=upperdash"`
        Bar                         `format:",inline"`
		Other string                `format:",inline,dateFormat=yyyy-MM-dd" json:"JsonCustomizedName"`
    }
)


func ExampleOfFormatTag() {
    tagValue :=  `format:",inline,dateFormat=yyyy-MM-dd" json:"JsonCustomizedName"`
    tag, err := format.Parse(tagValue, "json")
    if err != nil {
	    log.Fatal(err)	
    }
	fmt.Printf("%+v\n", tagValue)
}    
```


#### Parsing tags

```go


type Tag struct {
    Name string
	Setting1 string
	Repeated []string
	SettingN bool
	
}

func (t *Tag) updateTagKey(key value string) error {
    switch strings.ToLower(key) {
        case "name":
			t.Name = value
		case "setting1":
            t.Setting1 = value
        case "settingn":
			t.SettingN = value == "true" || value == ""
		case "repeated":
			t.Repeated = strigns.Split(strings.Tring(value, "{}", ",")
		default:
			return fmt.Errorf("unsupported tag: %s", key)
    }
	return nil
}


func ParseTag(tagString string) *Tag {
	tag := &Tag{}
		}
	if tagString == "-" {
		tag.Transient = true
	}
	values := tags.Values(tagString)
	name, values := values.Name()
	tag.Name = name
	_ = values.MatchPairs(tag.updateTagKey)
	return tag
}

```

##### format tag

Case formatter
```go
    
func ExampleCaseFormatter() {
	
    caseFormat := text.CaseFormatUpperUnderscore
    formatter := caseFormat.from.To(text.CaseFormatLowerCamel)
    formatted := formatter.Format("THIS_IS_TEST")
    fmt.Printf("formatted")

	
   detected := text.DetectCaseFormat("candidate", "candidate_2")
   fmt.Printf("detected: %s %v\n", detected, detected.IsDefined())
   
}

```

## Common tag:

- [fomat tag](format): tag to define output format 


## Contributing to tagly

tagly is an open source project and contributors are welcome!

See [TODO](TODO.md) list

## License

The source code is made available under the terms of the Apache License, Version 2, as stated in the file `LICENSE`.

Individual files may be made available under their own specific license,
all compatible with Apache License, Version 2. Please see individual files for details.


## Credits and Acknowledgements

**Library Author:** Adrian Witas

