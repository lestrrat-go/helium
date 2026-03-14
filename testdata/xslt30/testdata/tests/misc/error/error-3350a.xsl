<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<!--
      Error: It is a static error for a package to contain two or more non-hidden accumulators 
      with the same expanded QName and the same import precedence, unless there is another accumulator with 
      the same expanded QName, and a higher import precedence.
-->
   <xsl:template name="main">
      <out>
         <x/>
      </out>
   </xsl:template>
   
   <xsl:accumulator name="a" initial-value="0">
     <xsl:accumulator-rule match="x" select="$value+1"/>
   </xsl:accumulator>
   
   <xsl:accumulator name="a" initial-value="1">
     <xsl:accumulator-rule match="x" select="$value+2"/>
   </xsl:accumulator>
   
</xsl:stylesheet>
