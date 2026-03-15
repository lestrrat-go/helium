<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<!-- PURPOSE: test special case of format-number() imposing minWholePartSize = 1 -->

<xsl:template match="/">
<out>
<a int="{format-number(0, '#')}"/>
<a decimal="{format-number(0.0, '#')}"/>
<a double="{format-number(0.0e0, '#')}"/>
<a float="{format-number(xs:float(0), '#')}"/>
</out>
</xsl:template>

</xsl:stylesheet>
