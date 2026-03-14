<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

              <!--It is a dynamic error if the accumulator-before or accumulator-after function is called when the 
      context item is a node in a streamed document, unless the selected accumulator is declared with streamable="yes".
              -->
              
   <xsl:accumulator name="a" initial-value="0" streamable="false">
     <xsl:accumulator-rule match="text()" select="$value+1"/>
   </xsl:accumulator>
              
   <xsl:template name="main">
      <out>
         <xsl:source-document streamable="true" href="error-3360a.xsl">
            <xsl:value-of select="."/>
            <xsl:value-of select="accumulator-after('a')"/>
         </xsl:source-document>
      </out>
   </xsl:template>
</xsl:stylesheet>
