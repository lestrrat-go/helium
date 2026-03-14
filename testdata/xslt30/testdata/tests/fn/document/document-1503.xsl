<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Test that fragment identifiers in document() are stripped before
     determining document identity -->
  

  <xsl:template match="/">
    <out expect="1">
      <xsl:variable name="doc1" select="document('doc15.xml#frag1')"/>
      <xsl:variable name="doc2" select="document('doc15.xml#frag2')"/>
      <xsl:value-of select="count($doc1/ancestor-or-self::node()[last()] |
                            $doc2/ancestor-or-self::node()[last()])"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
