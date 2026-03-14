<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

   <!-- Using a static param in another static param in wrong decl order: ERROR scenario -->
   
   <xsl:param name="alt-param" static="yes" select="upper-case($static-param)"/>

   <xsl:param name="static-param" static="yes" select="'success'"/>

   <xsl:template name="xsl:initial-template" expand-text="yes">
      {$static-param}{$alt-param}
   </xsl:template>
   
</xsl:transform>
