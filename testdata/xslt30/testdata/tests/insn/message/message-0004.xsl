<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
				version="3.0">


	<xsl:template name="main">
   		<out xmlns:my="http://www.example.com/error">
   			<xsl:message terminate="yes" error-code="my:my-error-code">Yah boo</xsl:message>
   		</out>
	</xsl:template>

</xsl:stylesheet>
