<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:my="http://www.schemafunc.example.com/ns/various"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test fn:upper-case with arguments coming from typed nodes in input source. Types tested:
   				- xs:string, xs:normalizedString, xs:token, xs:language, xs:Name, xs:NCName, xs:NMTOKEN
   				xs:ID, xs:IDREF
   				- user-defined types derived by restriction from xs:string and all builtin types derived from it-->

   <xslt:import-schema namespace="http://www.schemafunc.example.com/ns/various"
      schema-location="variousTypesSchemaFunc.xsd"/>

   <xslt:template match="/my:userNode">
      <out>
         <e1>
            <xslt:value-of select="data(elem-string) instance of xs:string"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="upper-case(elem-string)"/>
         </e1>
         <e2>
            <xslt:value-of select="data(elem-normalizedString) instance of xs:normalizedString"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="upper-case(elem-normalizedString)"/>
         </e2>
         <e3>
            <xslt:value-of select="data(elem-token) instance of xs:token"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="upper-case(elem-token)"/>
         </e3>
         <e4>
            <xslt:value-of select="data(elem-language) instance of xs:language"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="upper-case(elem-language)"/>
         </e4>
         <e5>
            <xslt:value-of select="data(elem-Name) instance of xs:Name"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="upper-case(elem-Name)"/>
         </e5>
         <e6>
            <xslt:value-of select="data(elem-NCName) instance of xs:NCName"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="upper-case(elem-NCName)"/>
         </e6>
         <e7>
            <xslt:value-of select="data(elem-NMTOKEN) instance of xs:NMTOKEN"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="upper-case(elem-NMTOKEN)"/>
         </e7>
         <e8>
            <xslt:value-of select="data(elem-ID-attr/@attr1) instance of xs:ID"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="upper-case(elem-ID-attr/@attr1)"/>
         </e8>
         <e9>
            <xslt:value-of select="data(elem-IDREF-attr/@attr1) instance of xs:IDREF"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="upper-case(elem-IDREF-attr/@attr1)"/>
         </e9>
         <e10>
            <xslt:value-of select="data(elem-string-derived) instance of xs:string"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of
               select="data(elem-string-derived) instance of my:elem-string-derived-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="upper-case(elem-string-derived)"/>
         </e10>
         <e11>
            <xslt:value-of
               select="data(elem-normalizedString-derived) instance of xs:normalizedString"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of
               select="data(elem-normalizedString-derived) instance of my:elem-normalizedString-derived-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="upper-case(elem-normalizedString-derived)"/>
         </e11>
         <e12>
            <xslt:value-of select="data(elem-token-derived) instance of xs:token"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="data(elem-token-derived) instance of my:elem-token-derived-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="upper-case(elem-token-derived)"/>
         </e12>
         <e13>
            <xslt:value-of select="data(elem-language-derived) instance of xs:language"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of
               select="data(elem-language-derived) instance of my:elem-language-derived-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="upper-case(elem-language-derived)"/>
         </e13>
         <e14>
            <xslt:value-of select="data(elem-Name-derived) instance of xs:Name"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="data(elem-Name-derived) instance of my:elem-Name-derived-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="upper-case(elem-Name-derived)"/>
         </e14>
         <e15>
            <xslt:value-of select="data(elem-NCName-derived) instance of xs:NCName"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of
               select="data(elem-NCName-derived) instance of my:elem-NCName-derived-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="upper-case(elem-NCName-derived)"/>
         </e15>
         <e16>
            <xslt:value-of select="data(elem-NMTOKEN-derived) instance of xs:NMTOKEN"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of
               select="data(elem-NMTOKEN-derived) instance of my:elem-NMTOKEN-derived-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="upper-case(elem-NMTOKEN-derived)"/>
         </e16>
         <e17>
            <xslt:value-of select="data(elem-ID-attr-derived/@attr1) instance of xs:ID"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of
               select="data(elem-ID-attr-derived/@attr1) instance of my:id2-ID-length-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="upper-case(elem-ID-attr-derived/@attr1)"/>
         </e17>
         <e18>
            <xslt:value-of select="data(elem-IDREF-attr-derived/@attr1) instance of xs:IDREF"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of
               select="data(elem-IDREF-attr-derived/@attr1) instance of my:ir1-IDREF-enumeration-Type"/>
            <xslt:text> , </xslt:text>
            <xslt:value-of select="upper-case(elem-IDREF-attr-derived/@attr1)"/>
         </e18>
      </out>
   </xslt:template>
</xslt:transform>
