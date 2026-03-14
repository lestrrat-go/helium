<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test of xsl:next-match for two templates, one with pattern "element()" and 
  				the other one with pattern "element(*, xs:untyped)".-->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:template match="/doc">
      <out>
         <xslt:next-match/>
      </out>
   </xslt:template>

   <xslt:template match="element()">
	     <two xmlns:xsl="http://www.w3.org/1999/XSL/Transform">finally</two>
	     <xslt:next-match/>
   </xslt:template>

   <xslt:template match="element(*, xs:untyped)">
	     <one xmlns:xsl="http://www.w3.org/1999/XSL/Transform">one</one>
	     <xslt:next-match/>
   </xslt:template>
</xslt:transform>
