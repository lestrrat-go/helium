<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if 
                        within a package there is
                        more than one such declaration more than one
                              xsl:namespace-alias declaration with
                        the same literal namespace
                           URI and the same import
                           precedence and different values for the target namespace URI, unless
                        there is also an xsl:namespace-alias declaration with the
                        same literal namespace URI
                        and a higher import precedence. -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:out="http://output.ns/"
                version="2.0">
	
	<!-- TEST conflicting xsl:namespace declarations -->
	<?spec xslt#errors?><?error XTSE0810?>

	  <xsl:namespace-alias stylesheet-prefix="out" result-prefix="xsl"/>
	  <xsl:namespace-alias stylesheet-prefix="out" result-prefix="out"/>	
	  <xsl:template match="/">
		    <html out:version="1.0"/>
	  </xsl:template>
</xsl:stylesheet>
