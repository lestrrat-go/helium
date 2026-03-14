<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

   <!-- Req param with select attr: ERROR scenario -->
   
   <xsl:param name="req-param" static="yes" select="'this will fail with XTSE0010'" required="yes"/>

   <xsl:template name="xsl:initial-template" expand-text="yes">
      {$req-param}-{$req-param}
   </xsl:template>
   
</xsl:transform>
