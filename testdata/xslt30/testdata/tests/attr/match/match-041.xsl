<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version='2.0'>

<?spec xslt#patterns?>
<!-- test a pattern that includes range variable declarations in a predicate -->

  <xsl:template match="/">
    <out><xsl:apply-templates /></out>
  </xsl:template>
  
  <!-- default -->
  <xsl:template match="*">
    <xsl:apply-templates />
  </xsl:template>
  
  <!-- this is what causes the problem -->
  <xsl:template match="*[not(string-join(for $n in ancestor-or-self::* return concat('/', name($n), '[', string(count($n/preceding-sibling::*[name(.) = name($n)]) + 1), ']'), ''))]">
  </xsl:template>
 
  <xsl:template match="table">
     <xsl:call-template name="block-table" />
   </xsl:template>
   
   <xsl:template name="block-table">
     <xsl:variable name="testvar">the text to begin with</xsl:variable>
     <t>testvar: "<xsl:value-of select="$testvar" />"</t>
     <xsl:for-each select="./*">
       <xsl:apply-templates select="." />
       <t>testvar: "<xsl:value-of select="$testvar" />"</t>
     </xsl:for-each>
  </xsl:template> 

   <!-- Ignore All Text Nodes -->
   <xsl:template match="text()">     
   </xsl:template>
   
</xsl:stylesheet>
