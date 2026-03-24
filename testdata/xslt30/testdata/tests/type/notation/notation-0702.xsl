<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:one="http://notation.example.com"
             xmlns:two="http://test.notation.example.com"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="one two"
             version="2.0">
<!-- Purpose: Test 'ne' with two attributes of two different types derived from xs:NOTATION defined in separate schemas, 
  				with different targetNamespaces. Input document is validated against each, using the same prefix in both cases, 
  				but prefix bound to the appropriate namespace for each value.Returns true.-->

   <t:template match="/one:NOTATION-enumeration">
      <out>
         <t:value-of select="data(one:NOTATION-element/@NOTATION-attribute)"/>
         <t:text> * </t:text>
         <t:value-of select="data(two:doc/sub/@filetype)"/>
         <t:text> * </t:text>
         <t:value-of select="data(one:NOTATION-element/@NOTATION-attribute) ne data(two:doc/sub/@filetype)"/>
      </out>
   </t:template>
</t:transform>
