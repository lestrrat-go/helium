<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-id?>

  <xsl:template match="/">
  <out>
      <H1>Test for id().</H1>
      <HR/>
      <P>Should say "c":</P>
      <P><xsl:value-of select="id('c')/@id"/></P>
  </out>    
  </xsl:template>
 
</xsl:stylesheet>
