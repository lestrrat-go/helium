<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:one="http://www.defaultnamespace.example.com/ns/schema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs one"
                version="2.0">
<!-- Purpose: Test with xpath-default-namespace declared on the element that itself contains XPath expressions with 
  				type names inside @as and @type of an instruction.-->

   <xslt:import-schema namespace="http://www.defaultnamespace.example.com/ns/schema"
                       schema-location="schemaForDefaultnamespace.xsd"/>

   <xslt:template match="/">
      <out>
         <xslt:variable name="var1"
                        select="one:myPartNumberType('123-AB')"
                        as="myPartNumberType"
                        xpath-default-namespace="http://www.defaultnamespace.example.com/ns/schema"/>
         <xslt:variable name="var2"
                        select="doc-default"
                        as="schema-element(doc-default)"
                        xpath-default-namespace="http://www.defaultnamespace.example.com/ns/schema"/>
         <xslt:variable name="var3"
                        select="doc-default/elem"
                        as="element(*, myPartNumberType)"
                        xpath-default-namespace="http://www.defaultnamespace.example.com/ns/schema"/>
         <one>
            <xslt:value-of select="$var1 instance of one:myPartNumberType"/>
            <xslt:value-of select="$var2 instance of schema-element(one:doc-default)"/>
            <xslt:value-of select="$var3 instance of element(*, one:myPartNumberType)"/>
         </one>
         <two>
            <elem xslt:type="myPartNumberType" xslt:xpath-default-namespace="http://www.defaultnamespace.example.com/ns/schema">123-AB</elem>
         </two>
      </out>
   </xslt:template>
</xslt:transform>
