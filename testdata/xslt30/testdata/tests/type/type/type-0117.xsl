<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- PURPOSE: Test "instance of item()". -->
  <?spec xpath#id-instance-of?>
  <?spec xpath#id-matching-item?>

  <xsl:template match="/">
    <out>;
      <t><xsl:value-of select="1.0 instance of item()"/></t>;
      <t><xsl:value-of select="true() instance of item()"/></t>;
      <t><xsl:value-of select="'Paris' instance of item()"/></t>;
      <t><xsl:value-of select="1.0e5 instance of item()"/></t>;
      <t><xsl:value-of select="xs:float('1.0e5') instance of item()"/></t>;
      <t><xsl:value-of select="current-dateTime() instance of item()"/></t>;         
    </out>
  </xsl:template>

</xsl:stylesheet>