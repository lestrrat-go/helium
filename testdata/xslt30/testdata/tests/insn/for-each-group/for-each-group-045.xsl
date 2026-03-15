<?spec xslt#grouping?>
<xsl:transform 
  version="3.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  <!-- in 3.0, group adjacent using atomic keys including NaNs -->
  
<xsl:template match="/">
<out>
	<xsl:for-each-group select="1 to 30" 
	                    group-adjacent=". idiv 2, number(concat(., 'zzz'))"
	                    composite="1">
	  <group values="{current-group()}"/>
	</xsl:for-each-group>
</out>
</xsl:template>
	
</xsl:transform>