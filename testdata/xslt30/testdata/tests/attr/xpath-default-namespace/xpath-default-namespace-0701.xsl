<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:d="http://www.defaultnamespace.example.com/ns/schema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs d"
                version="2.0">

<!-- Testcase with @xpath-default-namespace on the parent of a variable 
     that has an unprefixed element name or type name in an 'as' attribute.-->

   <xslt:import-schema namespace="http://www.defaultnamespace.example.com/ns/schema"
                       schema-location="schemaForDefaultnamespace.xsd"/>

   <xslt:template match="/">
      <out xslt:xpath-default-namespace="http://www.defaultnamespace.example.com/ns/schema">
         <xslt:variable name="v1"
                        select="d:myPartNumberType('123-AB')"
                        as="myPartNumberType"/>
         <xslt:variable name="v2" select="doc-default" as="schema-element(doc-default)"/>
         <xslt:variable name="v3"
                        select="doc-default/elem"
                        as="element(*, myPartNumberType)"/>
         <xslt:value-of select="$v1 instance of d:myPartNumberType"/>
         <xslt:value-of select="$v2 instance of schema-element(d:doc-default)"/>
         <xslt:value-of select="$v3 instance of element(*, myPartNumberType)"/>
      </out>
   </xslt:template>
</xslt:transform>
