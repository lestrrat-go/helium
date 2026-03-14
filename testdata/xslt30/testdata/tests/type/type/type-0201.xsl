<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

  <!-- PURPOSE: Test "instance of" on node types. -->
  <?spec xpath#id-instance-of?>
  <?spec xpath#id-matching-item?>

  <xsl:template match="/">
    <out changed="2004-03-05">;
      <xsl:for-each select="doc/child::node()">
      <p position="{position()}">
          <t><xsl:value-of select=". instance of element()"/></t>;
          <t><xsl:value-of select=". instance of attribute()"/></t>;
          <t><xsl:value-of select=". instance of comment()"/></t>;
          <t><xsl:value-of select=". instance of processing-instruction()"/></t>;
          <t><xsl:value-of select=". instance of document-node()"/></t>;
          <t><xsl:value-of select=". instance of text()"/></t>;
          <!--<t><xsl:value-of select=". instance of namespace()"/></t>;-->    
          <t><xsl:value-of select=". instance of node()"/></t>;              
          <t><xsl:value-of select=". instance of item()"/></t>;                                                            
      </p>
      </xsl:for-each>
    </out>
  </xsl:template>

</xsl:stylesheet>