<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test of xsl:next-match in a template where the pattern has a union 
  				operator where both branches of the patternmatch one and the same node. -->
   <!-- Same as next-match-021 except that the two branches of the union have the same priority.
        Technically, the second pattern should be matched before the first -->

   <xslt:output method="xml" encoding="UTF-8" indent="no"/>

   <xslt:template match="doc">
      <out>
         <xslt:apply-templates select="one"/>
      </out>
   </xslt:template>

   <xslt:template match="one[xs:integer(.) gt 0] | one[starts-with(., '1')]">
	     <one xmlns:xsl="http://www.w3.org/1999/XSL/Transform">one time</one>
	     <xslt:next-match/>
   </xslt:template>

   <xslt:template match="node()">
	     <two xmlns:xsl="http://www.w3.org/1999/XSL/Transform">finally</two>
   </xslt:template>
</xslt:transform>
