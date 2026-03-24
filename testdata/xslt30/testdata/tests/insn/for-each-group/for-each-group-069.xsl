<?spec xslt#grouping?>
<xsl:transform 
  version="3.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">
  
  <!-- in 3.0, group using variable-length composite keys -->
  
<xsl:template match="/">
<out>
	<xsl:for-each-group select="//*" 
	                    group-by="ancestor-or-self::*/name()"
	                    composite="yes">
	  <element path="{string-join(current-grouping-key(), '/')}" count="{count(current-group())}"/>
	</xsl:for-each-group>
</out>
</xsl:template>
	
</xsl:transform>