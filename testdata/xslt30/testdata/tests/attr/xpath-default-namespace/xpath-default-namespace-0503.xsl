<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Testcase with @xpath-default-namespace on the parent of an instruction that has an unprefixed type name in an 'type' attribute.-->

   <xslt:import-schema namespace="http://www.defaultnamespace.example.com/ns/schema"
                       schema-location="schemaForDefaultnamespace.xsd"/>

   <xslt:template match="/">
      <out xslt:xpath-default-namespace="http://www.defaultnamespace.example.com/ns/schema">
         <elem xslt:type="myPartNumberType">123-AB</elem></out>
   </xslt:template>
</xslt:transform>
