<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<!-- PURPOSE:  Type error in argument to arithmetic operator, works in
               backwards compatible mode -->
               <?spec xslt#backwards?>
               
               <!-- for failing example with version=2.0, see error229 -->

<xsl:template match="doc">
<out>
<a val="{string(3 + '2')}" xsl:version="1.0"/>;
</out>
</xsl:template>
</xsl:stylesheet>