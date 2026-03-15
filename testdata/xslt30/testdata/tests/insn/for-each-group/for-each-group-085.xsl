<xsl:stylesheet version="2.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="#all">
  
  <xsl:template match="/">
    <out>
      <xsl:apply-templates/>
    </out>
  </xsl:template>
  
  <xsl:template match="/c">
    <xsl:variable name="c" as="element()*" select="*[2]"/>
    
    <xsl:variable name="t" as="element()*">
      <tt n="T"/>
      <tt n="U"/>
      <tt n="T"/>
      <tt n="U"/>
      <tt n="T"/>
    </xsl:variable>
    
    <xsl:variable name="o" as="element()*">
      <xsl:for-each-group select="$t" group-by="@n">
        <xsl:variable name="g" as="element()+" 
          select="current-group()"/>
        
        <xsl:sequence select="$g[@n = $c/@n]"/>
      </xsl:for-each-group>
    </xsl:variable>
    
    <xsl:apply-templates>
      <xsl:with-param name="o" tunnel="yes" select="$o"/>
    </xsl:apply-templates>
  </xsl:template>
  
  <xsl:template match="t">
    <xsl:param name="o" tunnel="yes" as="element()*"/>
    <xsl:copy>
      <xsl:copy-of select="@*"/>
      <xsl:copy-of select="$o"/>
    </xsl:copy>   
  </xsl:template>
  
</xsl:stylesheet>