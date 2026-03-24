<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <!-- current-output-uri() cannot be used in static expressions -->
  
  <xsl:template name="xsl:initial-template">
      <xsl:result-document _href="{current-output-uri()}">
          <out><!-- error --></out>
      </xsl:result-document>
  </xsl:template>

</xsl:stylesheet>
