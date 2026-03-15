<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    package-version="1.0-valid"
    _package-version="{if(number(system-property('xsl:version')) ge 3 ) then '' else '4.0'}"
    version="3.0">
    
    <!-- the shadow attribute makes it empty, which is not allowed -->
    
    <xsl:template name="xsl:initial-template">SHOULD NOT FIRE</xsl:template>

</xsl:package>