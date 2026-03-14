<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

                <!--It is a dynamic error if the
                    current-grouping-key function is used when the
                    current grouping key is absent, or when it is invoked
                    in the course of evaluating a pattern..
                    The error may be reported statically if it can be detected statically.-->
                    
   <xsl:template name="main">
      <out>
         <xsl:value-of select="current-grouping-key()"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
