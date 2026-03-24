<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xslt#key?>
<!-- PURPOSE: test a key with a numeric predicate in the match pattern 
     ([1] is treated specially in Saxon) -->

<xsl:key name="k" match="item[1]" use="s"/>

  <xsl:template match="/">
      <out>
      <xsl:for-each select="key('k', '22')">
         <xsl:copy-of select="."/>;
      </xsl:for-each>
      </out>
  </xsl:template>
 
</xsl:stylesheet>
