<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

      <!--It is a non-recoverable dynamic error if an element has an effective version of V (with
          V < 3.0) when the implementation does not support
          backwards compatible behavior for XSLT version V. -->
          
   <xsl:template name="main">
      <out>
         <x xsl:version="1.0"/>
      </out>
   </xsl:template>
</xsl:stylesheet>
