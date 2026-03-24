<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test that the result of 'ne' on xs:NOTATION type arguments is of type xs:boolean. 
  				The schema has no @targetNamespace. -->

   <xslt:import-schema schema-location="noNamespaceNotation.xsd"/>

   <xslt:variable name="v" select="data(/doc/sub/@filetype)"/>

   <xslt:template match="/">
      <out>
         <xslt:value-of select="$v instance of nota"/>
         <xslt:value-of select="$v instance of xs:NOTATION"/>
         <xslt:value-of select="($v ne doc/sub/@filetype) instance of xs:boolean"/>
      </out>
   </xslt:template>
</xslt:transform>
