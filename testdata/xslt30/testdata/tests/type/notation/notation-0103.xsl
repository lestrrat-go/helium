<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test 'eq' and 'ne' on one and the same attribute value. The schema has no @targetNamespace. -->

   <xslt:import-schema schema-location="noNamespaceNotation.xsd"/>

   <xslt:template match="/">
      <out>
         <xslt:value-of select="data(/doc/sub/@filetype) instance of xs:NOTATION"/>
         <xslt:value-of select="/doc/sub/@filetype eq doc/sub/@filetype"/>
         <xslt:value-of select="/doc/sub/@filetype ne doc/sub/@filetype"/>
      </out>
   </xslt:template>
</xslt:transform>
