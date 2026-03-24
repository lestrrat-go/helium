<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- PURPOSE: Test "cast as" on primitive types. -->
  <?spec xpath#id-cast?>
  <?spec xpath#id-predefined-types?>

  <xsl:template match="/">
    <out>;
      <t><xsl:value-of select="(3+2) cast as xs:decimal "/></t>;
      <t><xsl:value-of select="('false') cast as xs:boolean "/></t>;
      <t><xsl:value-of select="(3+2) cast as xs:string "/></t>;
      <t><xsl:value-of select=" (5*1000) cast as xs:double"/></t>;
      <t><xsl:value-of select="(100000000) cast as xs:float "/></t>;
      <t><xsl:value-of select="('2002-02-20T12:00:00') cast as xs:dateTime"/></t>;         
    </out>
  </xsl:template>

</xsl:stylesheet>