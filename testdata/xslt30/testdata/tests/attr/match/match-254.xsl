<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="#all">
    
    <!-- A large set of patterns using both a predicate and a parent qualifier -->
    
    <xsl:template match="doc[@foo]/a[@code='a']">1</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='b']">2</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='c']">3</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='d']">4</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='e']">5</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='f']">6</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='g']">7</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='h']">8</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='i']">9</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='j']">10</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='k']">11</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='l']">12</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='m']">13</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='n']">14</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='o']">15</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='p']">16</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='q']">17</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='r']">18</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='s']">19</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='t']">20</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='u']">21</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='v']">22</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='w']">23</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='x']">24</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='y']">25</xsl:template>
    <xsl:template match="doc[@foo]/a[@code='z']">26</xsl:template>
    <xsl:template match="a">0</xsl:template>
    
    <xsl:variable name="data">
        Unfortunately the attempt to fix this was done simply by modifying the generated tests. 
        This test set must not be manually modified, or the changes will simply disappear the next 
        time the spec is rebuilt. The problem needs to be fixed at source by generating correct 
        tests from the examples in the spec.
    </xsl:variable>
    
    <xsl:template name="main">
        <out>
            <xsl:for-each select="string-to-codepoints(normalize-space($data))!codepoints-to-string(.)">
                <xsl:variable name="e" as="element(doc)">
                    <doc foo="3"><a code="{.}"/></doc>
                </xsl:variable>
                <xsl:variable name="n" as="xs:integer">
                    <xsl:apply-templates select="$e/a"/>
                </xsl:variable>
                <xsl:sequence select="$n"/>
            </xsl:for-each>
        </out>
    </xsl:template>
    
 
</xsl:stylesheet>
