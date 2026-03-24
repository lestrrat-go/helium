<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

                <!--    When the containing element is processed with XSLT 1.0
                        behavior, it is a non-recoverable dynamic error to
                        evaluate an extension function call if no implementation of the extension
                        function is available.-->
                        
   <xsl:template name="main">
      <out>
         <xsl:value-of select="ext:f(3)" xmlns:ext="http://example.com/functions" version="1.0"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
