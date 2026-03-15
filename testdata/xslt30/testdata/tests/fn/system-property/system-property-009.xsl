<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:t="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test to verify that all specification defined arguments of system-property must be lower case or they will not be recognized. -->

   <t:template match="/">
      <out>
         <t:value-of select="system-property('t:VERSION')"/>
         <t:value-of select="system-property('t:VENDOR')"/>
         <t:value-of select="system-property('t:VENDOR-URL')"/>
         <t:value-of select="system-property('t:PRODUCT-NAME')"/>
         <t:value-of select="system-property('t:PRODUCT-VERSION')"/>
         <t:value-of select="system-property('t:IS-SCHEMA-AWARE')"/>
         <t:value-of select="system-property('t:SUPPORTS-SERIALIZATION')"/>
         <t:value-of select="system-property('t:SUPPORTS-BACKWARDS-COMPATIBILITY')"/>
      </out>
   </t:template>
</t:transform>
