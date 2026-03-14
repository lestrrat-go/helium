<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

           <!--It is a dynamic error if an xsl:evaluate
               instruction is evaluated when use of xsl:evaluate is disabled.
            -->
            
   <xsl:template name="main">
      <out>
         <xsl:evaluate xpath="'1 + 2'" />
      </out>
   </xsl:template>
   
</xsl:stylesheet>
