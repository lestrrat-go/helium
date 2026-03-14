<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: test optimisation of a filter expression containing a
       sequence-valued non-dependent subexpression  -->
       <?spec xpath#id-predicates?>
  <xsl:template match="doc">
  <xsl:param name="p" select="5"/>
    <out>;
        <xsl:variable name="aaa" select="a"/>
        <xsl:variable name="seq" select="a[position()&gt;2]"/>
        <xsl:value-of select="$seq[local-name()=$aaa/@att]"/>
    </out>
  </xsl:template>
 
</xsl:stylesheet>
