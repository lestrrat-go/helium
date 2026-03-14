<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- PURPOSE: Test "instance of" on primitive types. -->
  <?spec xpath#id-instance-of?>
  <?spec xpath#id-predefined-types?>

  <xsl:template match="/">
    <out>;
      <t><xsl:value-of select="1.0 instance of xs:decimal"/></t>;
      <t><xsl:value-of select="true() instance of xs:boolean"/></t>;
      <t><xsl:value-of select="'Paris' instance of xs:string"/></t>;
      <t><xsl:value-of select="1.0e5 instance of xs:double"/></t>;
      <t><xsl:value-of select="xs:float('1.0e5') instance of xs:float"/></t>;
      <t><xsl:value-of select="current-dateTime() instance of xs:dateTime"/></t>;         
    </out>
  </xsl:template>

</xsl:stylesheet>