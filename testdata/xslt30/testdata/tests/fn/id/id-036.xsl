<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec fo#func-id?>
  <xsl:template match="/">
    <out>
      <H1>Test for id().</H1>
      <P>Only worked with the XML4J 1.1.16 liaison at one time!</P>
      <HR/>
      <P>Should say "c":</P>
      <P><xsl:value-of select="id('c')/@id"/></P>
      <xsl:apply-templates/>
    </out>  
  </xsl:template>
 
</xsl:stylesheet>
