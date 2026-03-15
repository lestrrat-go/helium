<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="1.0"
                xmlns:baz1="http://xsl.lotus.com/ns1"
                xmlns:baz2="http://xsl.lotus.com/ns2">

<?spec fo#func-namespace-uri?>
<!-- PURPOSE:   Test of 'namespace-uri()'. -->

<!-- note, this relies on namespace-uri taking the first element in the sequence, which
     only works under backwards-compatible mode -->
     
<xsl:template match="baz2:doc">
	<out>
	   <xsl:value-of select="namespace-uri(*)"/>
	</out>
</xsl:template>
 
</xsl:stylesheet>
