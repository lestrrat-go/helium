<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                                 error if the result sequence contains a namespace node
                              with no name and the element node being constructed has a null
                              namespace URI (that is, it is an error to define a default namespace
                              when the element is in no namespace). -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
	
	<!-- TEST xsl:namespace declaring the default namespace (XSLT 2.0) -->
	<!-- this is an error if the element is in no namespace -->
	<?spec xslt#errors?><?error XTDE0440?>
	
	  <xsl:param name="param" select="'http://a-namespace-uri/'"/>

	  <xsl:template match="/">
		    <out xmlns:x="http://some.uri/">
		       <xsl:namespace name="">
            <xsl:value-of select="$param"/>
         </xsl:namespace>
		       <xsl:attribute name="new">at XSLT 2.0</xsl:attribute>
		    </out>
	  </xsl:template>

</xsl:stylesheet>
