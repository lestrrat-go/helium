<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

            <!--It is a non-recoverable dynamic
                error if the invocation of the stylesheet specifies 
                both an initial mode and an initial template.
            -->
            
   <xsl:template name="main">
      <out>
         <x/>
      </out>
   </xsl:template>
   
   <xsl:mode name="m"/>
</xsl:stylesheet>
