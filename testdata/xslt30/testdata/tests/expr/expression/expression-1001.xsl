<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
>

<!-- Test intersection and difference -->
<?spec xpath#combining_seq?>

<xsl:variable name="i" select="//city[contains(@name,'i')]"/>
<xsl:variable name="e" select="//city[contains(@name,'e')]"/>

<xsl:template match="/">
  <out>
    Containing i and e:
    <xsl:for-each select="$i intersect $e">
         <xsl:value-of select="@name"/>;
    </xsl:for-each>     
    Containing i and no e:
    <xsl:for-each select="$i except $e">
         <xsl:value-of select="@name"/>;
    </xsl:for-each> 
    Containing e and no i:
    <xsl:for-each select="$e except $i">
         <xsl:value-of select="@name"/>;
    </xsl:for-each> 
  </out>
</xsl:template>
 
</xsl:stylesheet>
