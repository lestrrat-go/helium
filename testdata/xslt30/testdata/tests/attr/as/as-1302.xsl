<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://www.mytest.net"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my" version="2.0">
   <!-- Purpose: Test with global xsl:variable that has @as="element(*, xs:untyped?)+" 
				and @select contains a sequence of element nodes from input file.-->

   <xslt:variable name="var1" select="/doc//*" as="element(*, xs:untyped?)+"/>

   <xslt:template match="/doc">
      <out>
         <e1>
            <xslt:value-of select="$var1 instance of element(*, xs:untyped?)+"/>
            <xslt:value-of select="$var1 instance of element(*, xs:untyped?)"/>
         </e1>
      </out>
   </xslt:template>
</xslt:transform>
