<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

   <!-- Req param with select attr: ERROR scenario -->
   
   <xsl:param name="wrong-param" static="yes">
      This will raise XTSE0010
   </xsl:param>

   <xsl:template name="xsl:initial-template" expand-text="yes">
      {$wrong-param}
   </xsl:template>
   
</xsl:transform>
