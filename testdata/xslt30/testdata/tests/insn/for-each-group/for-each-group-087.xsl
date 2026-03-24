<?spec xslt#grouping?>
<xsl:transform 
  version="3.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  <!-- in 3.0, group using composite keys with a case-blind collation -->
  
  <xsl:param name="collation" as="xs:string" select="'http://www.w3.org/2013/collation/UCA?strength=secondary'"/>
  
<xsl:template match="/">
<out>
	<xsl:for-each-group select="/*/city" 
	                    group-by="@country, xs:decimal(@pop)"
	                    composite="yes"
	                    collation="{$collation}">
	  <group country="{lower-case(current-grouping-key()[1])}" pop="{current-grouping-key()[2]}">
	    <xsl:apply-templates select="current-group()"/>
	  </group>
	</xsl:for-each-group>
</out>
</xsl:template>
  
<xsl:template match="city">
  <xsl:copy>
    <xsl:apply-templates select="@*"/>
  </xsl:copy>
</xsl:template>
  
<xsl:template match="@name|@pop">
  <xsl:copy select="."/>
</xsl:template>
  
  <xsl:template match="@country">
    <xsl:attribute name="country" select="lower-case(.)"/>
  </xsl:template>
	
</xsl:transform>