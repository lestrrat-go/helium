<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

             <!--It is a dynamic error if the
                 accumulator-before or accumulator-after function for an accumulator A
                 is evaluated in the course of evaluating the select
                 expression or contained sequence constructor of any accumulator rule for the same accumulator A.-->
                 
   <xsl:accumulator name="a" initial-value="0">
     <xsl:accumulator-rule match="*" select="$value + accumulator-before('a')"/>
   </xsl:accumulator>
   
   <xsl:variable name="v"><x/></xsl:variable>
   
   <xsl:template name="main">
      <out>
         <xsl:for-each select="$v/x">
           <xsl:value-of select="accumulator-before('a')"/>
         </xsl:for-each>
      </out>
   </xsl:template>
</xsl:stylesheet>
