<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="#all">
    
    <!-- A large pattern set using integer comparisons -->
    
    <xsl:template match="a[number(@code) eq 1]">one</xsl:template>
    <xsl:template match="a[number(@code) eq 2]">two</xsl:template>
    <xsl:template match="a[number(@code) eq 3]">three</xsl:template>
    <xsl:template match="a[number(@code) eq 4]">four</xsl:template>
    <xsl:template match="a[number(@code) eq 5]">five</xsl:template>
    <xsl:template match="a[number(@code) eq 6]">six</xsl:template>
    <xsl:template match="a[number(@code) eq 7]">seven</xsl:template>
    <xsl:template match="a[number(@code) eq 8]">eight</xsl:template>
    <xsl:template match="a[number(@code) eq 9]">nine</xsl:template>
    <xsl:template match="a[number(@code) eq 10]">ten</xsl:template>
    <xsl:template match="a[number(@code) eq 11]">eleven</xsl:template>
    <xsl:template match="a[number(@code) eq 12]">twelve</xsl:template>
    <xsl:template match="a[number(@code) eq 13]">thirteen</xsl:template>
    <xsl:template match="a[number(@code) eq 14]">fourteen</xsl:template>
    <xsl:template match="a[number(@code) eq 15]">fifteen</xsl:template>
    <xsl:template match="a[number(@code) eq 16]">sixteen</xsl:template>
    <xsl:template match="a[number(@code) eq 17]">seventeen</xsl:template>
    <xsl:template match="a[number(@code) eq 18]">eighteen</xsl:template>
    <xsl:template match="a[number(@code) eq 19]">nineteen</xsl:template>
    <xsl:template match="a[number(@code) eq 20]">twenty</xsl:template>
    <xsl:template match="a[number(@code) eq 21]">twenty-one</xsl:template>
    <xsl:template match="a[number(@code) eq 22]">twenty-two</xsl:template>
    <xsl:template match="a[number(@code) eq 23]">twenty-three</xsl:template>
    <xsl:template match="a[number(@code) eq 24]">twenty-four</xsl:template>
    <xsl:template match="a[number(@code) eq 25]">twenty-five</xsl:template>
    <xsl:template match="a[number(@code) eq 26]">twenty-six</xsl:template>
    <xsl:template match="a[number(@code) eq 27]">twenty-seven</xsl:template>
    <xsl:template match="a[number(@code) eq 28]">twenty-eight</xsl:template>
    <xsl:template match="a[number(@code) eq 29]">twenty-nine</xsl:template>
    <xsl:template match="a[number(@code) eq 30]">thirty</xsl:template>
    
  
    
    <xsl:template name="main">
        <out>
            <xsl:for-each select="1 to 30">
                <xsl:variable name="e" as="element(a)">
                    <a code="{.}"/>
                </xsl:variable>
                <xsl:apply-templates select="$e"/>
                <xsl:text>,</xsl:text>
            </xsl:for-each>
        </out>
    </xsl:template>
    
 
</xsl:stylesheet>
