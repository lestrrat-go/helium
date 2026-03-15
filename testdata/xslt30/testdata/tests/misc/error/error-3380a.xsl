<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

              <!--ORIGINAL TEST (NOW SUPERSEDED):
              
                  If the accumulator is declared with the attribute streamable="yes"
                  then it is a dynamic error if the accumulator-before or
                  accumulator-after function is called unless the evaluation
                  is performed in the course of the evaluation of either (a) the
                  sequence constructor contained in a template rule invoked in a mode 
                  declared with streamable="yes", or (b) the
                  sequence constructor contained in an xsl:source-document instruction, or
                  both. -->
                  
   <xsl:accumulator name="a" streamable="yes" initial-value="0">
     <xsl:accumulator-rule match="*" select="$value+1"/>
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
