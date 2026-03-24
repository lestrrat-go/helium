<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

   <!-- Tests static attribute on xsl:param and dynamic ctx usage in a TVT -->
   
   <xsl:param name="static-param" static="yes" select="'success'"/>
   
   <xsl:template name="xsl:initial-template" expand-text="yes">
      {$static-param}
   </xsl:template>
   
</xsl:transform>
