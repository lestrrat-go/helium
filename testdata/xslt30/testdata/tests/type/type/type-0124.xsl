<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- PURPOSE: Test "instance of" on a temporary tree. -->
  <?spec xpath#id-instance-of?>
  <?spec xpath#id-element-test?>
  <?spec xpath#id-matching-item?>
  
<xsl:variable name="data">
  <a>2002-02-02T12:00:00</a>
  <a>2002-02-03T12:00:00</a>
</xsl:variable>

<xsl:variable name="data2">z</xsl:variable>

  <xsl:template match="/">
    <out>;
      <t><xsl:value-of select="$data/a instance of element()+"/></t>;
      <t><xsl:value-of select="$data2 instance of document-node()"/></t>;
      <t><xsl:value-of select="$data instance of node()"/></t>;
      <t><xsl:value-of select="$data2 instance of item() *"/></t>;            
    </out> 
  </xsl:template>

</xsl:stylesheet>