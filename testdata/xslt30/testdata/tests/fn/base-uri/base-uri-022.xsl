<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                xml:base="http://www.baseuri.exmpl/tests/"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Testcase with fn:base-uri($arg) to verify the  result if of type xs:anyURI?. 
  				Includes test with document node.-->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:template match="/">
      <out>
         <xslt:value-of select="base-uri(.)  instance of xs:anyURI?"/>
         <xslt:value-of select="base-uri(//str1) instance of xs:anyURI?"/>
      </out>
   </xslt:template>
</xslt:transform>
