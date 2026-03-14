<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

   <!-- Static param without select attr, defaults to empty string -->
   
   <xsl:param name="static-param" static="yes" />

   <xsl:template name="xsl:initial-template" expand-text="yes">
      {empty($static-param)}-{upper-case($static-param)}-{$static-param = ''}
   </xsl:template>
   
</xsl:transform>
