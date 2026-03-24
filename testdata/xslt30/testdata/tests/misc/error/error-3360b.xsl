<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

              <!--It is a type error to call the accumulator-before or
                  accumulator-after function when the context item is not a node, or when it is an attribute or namespace
                  node. 
              -->
              
   <xsl:accumulator name="a" initial-value="0">
     <xsl:accumulator-rule match="text()" select="$value+1"/>
   </xsl:accumulator>
              
   <xsl:template name="main">
      <out>
         <xsl:for-each select="1 to 10">
            <xsl:value-of select="accumulator-after('a')"/>
         </xsl:for-each>
      </out>
   </xsl:template>
</xsl:stylesheet>
