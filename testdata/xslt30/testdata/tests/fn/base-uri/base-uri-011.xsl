<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                xml:base="http://www.baseuri.exmpl/tests/"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Testcase with fn:base-uri to verify the  result is of type xs:anyURI?.-->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:template match="/">
      <out>
         <xslt:value-of select="base-uri() instance of xs:anyURI?"/>
      </out>
   </xslt:template>
</xslt:transform>
