<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xsd="http://www.w3.org/2001/XMLSchema">

  <!-- PURPOSE: Test "instance of" using xsd namespace. -->
  <?spec xpath#id-instance-of?>
  <?spec xpath#id-predefined-types?>

  <xsl:template match="/">
    <out>;
      <t><xsl:value-of select="1.0 instance of xsd:decimal"/></t>;
      <t><xsl:value-of select="true() instance of xsd:boolean"/></t>;
      <t><xsl:value-of select="'Paris' instance of xsd:string"/></t>;
      <t><xsl:value-of select="1.0e5 instance of xsd:double"/></t>;
      <t><xsl:value-of select="xsd:float('1.0e5') instance of xsd:float"/></t>;
      <t><xsl:value-of select="current-dateTime() instance of xsd:dateTime"/></t>;         
    </out>
  </xsl:template>

</xsl:stylesheet>