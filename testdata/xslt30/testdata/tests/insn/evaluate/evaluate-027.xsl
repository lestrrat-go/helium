<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="3.0" 
				xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
				xmlns:xs="http://www.w3.org/2001/XMLSchema"
				xmlns="http://www.w3.org/1999/xhtml"
				exclude-result-prefixes="xs #default"
				xpath-default-namespace="http://unknown.uri/should/not/interfere/with/test">
				
	<!-- test xsl:evaluate using a second argument supplying a namespace context -->
				
	<xsl:template match="/">
        <out>
            
        	<xsl:variable name="context" as="element()">
              <a:a xmlns:a="http://a.uri/" xmlns:b="http://b.uri/" xmlns:c="http://c.uri/">
                <xsl:namespace name="">http://d.uri/</xsl:namespace>
              </a:a>
            </xsl:variable>
        	
            <xsl:evaluate xpath="'/a:a/b:b/c:c/d/string()'"
                namespace-context="$context"
                context-item="/"/>
        </out>
	</xsl:template>
	
</xsl:stylesheet>
