<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

                <!--It is a dynamic error if the
                    current-grouping-key function is used when the
                    current grouping key is absent, or when it is invoked
                    in the course of evaluating a pattern..
                    The error may be reported statically if it can be detected statically.-->
                    
   <xsl:template name="main">
      <out>
         <xsl:for-each-group select="1 to 50" group-starting-with=".[. mod 10 = 0]">
            <xsl:value-of select="current-grouping-key()"/>
         </xsl:for-each-group>   
      </out>
   </xsl:template>
</xsl:stylesheet>
