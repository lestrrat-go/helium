<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">
<!-- Purpose: Test with xsl:template which returns a sequence of several explicitly 
  				constructed different built-in types and @as="xs:anyAtomicType+".  -->

   <xslt:output method="xml" encoding="UTF-8"/>

   <xslt:template match="/doc">
      <out>
         <xslt:call-template name="temp1"/>
      </out>
   </xslt:template>

   <xslt:template name="temp1" as="xs:anyAtomicType+">
	     <xslt:sequence select="(xs:duration('P1Y2M3DT10H30M23S'), xs:dateTime('2000-01-16T00:00:00Z'), xs:gYear('2005'), abs(xs:double(-0.0E0)))"/>
   </xslt:template>
</xslt:transform>
