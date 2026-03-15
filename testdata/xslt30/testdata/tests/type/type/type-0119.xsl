<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xsd="http://www.w3.org/2001/XMLSchema">

  <!-- PURPOSE: Test "instance of" returning false. -->
  <?spec xpath#id-instance-of?>
  <?spec xpath#id-predefined-types?>

  <xsl:template match="/">
    <out>;
      <f><xsl:value-of select="1.0 instance of xsd:string"/></f>;
      <f><xsl:value-of select="true() instance of xsd:decimal"/></f>;
      <f><xsl:value-of select="'Paris' instance of xsd:float"/></f>;
      <f><xsl:value-of select="1.0e5 instance of xsd:anyURI"/></f>;
      <f><xsl:value-of select="xsd:float('1.0e5') instance of xsd:dateTime"/></f>;
      <f><xsl:value-of select="current-dateTime() instance of node()"/></f>;         
    </out>
  </xsl:template>

</xsl:stylesheet>