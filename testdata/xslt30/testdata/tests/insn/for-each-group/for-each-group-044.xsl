<?spec xslt#grouping?>
<xsl:transform 
  version="3.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  <!-- in 3.0, group adjacent using composite keys -->
  
<xsl:template match="/">
<out>
	<xsl:for-each-group select="/*/city" 
	                    group-adjacent="@country, xs:decimal(@pop)"
	                    composite="true">
	  <group country="{current-grouping-key()[1]}" pop="{current-grouping-key()[2]}">
	    <xsl:copy-of select="current-group()"/>
	  </group>
	</xsl:for-each-group>
</out>
</xsl:template>
	
</xsl:transform>