<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
    <!-- basic tests with xsl:on-empty, nested in variables -->
  
    <xsl:template name="xsl:initial-template">
        <xsl:variable name="empty">
            <empty>
                <xsl:variable name="empty">
                    <xsl:variable name="empty">
                        <empty>
                            <!-- kicks in, creates <empty>42</empty> -->
                            <xsl:on-empty select="42"/> 
                        </empty>
                        <!-- does not kick in -->
                        <xsl:on-empty select="42"/> 
                    </xsl:variable>
                    <!-- returns <empty>42</empty> -->
                    <xsl:sequence select="$empty/empty" />
                    <!-- does not kick in -->
                    <xsl:on-empty select="42"/> 
                </xsl:variable>
                <!-- returns <empty>42</empty> -->
                <xsl:sequence select="$empty/empty" />
                <!-- does not kick in -->
                <xsl:on-empty select="42"/> 
            </empty>
        </xsl:variable>
        <!-- returns <empty><empty>42</empty></empty> -->
        <xsl:sequence select="$empty/empty" />
        <!-- does not kick in -->
        <xsl:on-empty select="42"/>
    </xsl:template>

</xsl:stylesheet>
