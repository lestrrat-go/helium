<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="#all">
    
    <xsl:template match="a[@code='unfortunately']">1</xsl:template>
    <xsl:template match="a[@code='the']">2</xsl:template>
    <xsl:template match="a[@code='attempt']">3</xsl:template>
    <xsl:template match="a[@code='to']">4</xsl:template>
    <xsl:template match="a[@code='fix']">5</xsl:template>
    <xsl:template match="a[@code='this']">6</xsl:template>
    <xsl:template match="a[@code='was']">7</xsl:template>
    <xsl:template match="a[@code='done']">8</xsl:template>
    <xsl:template match="a[@code='simply']">9</xsl:template>
    <xsl:template match="a[@code='by']">10</xsl:template>
    <xsl:template match="a[@code='modifying']">11</xsl:template>
    <xsl:template match="a[@code='generated']">12</xsl:template>
    <xsl:template match="a[@code='tests']">13</xsl:template>
    <xsl:template match="a[@code='test']">14</xsl:template>
    <xsl:template match="a[@code='set']">15</xsl:template>
    <xsl:template match="a[@code='must']">16</xsl:template>
    <xsl:template match="a[@code='not']">17</xsl:template>
    <xsl:template match="a[@code='manually']">18</xsl:template>
    <xsl:template match="a[@code='modified']">19</xsl:template>
    <xsl:template match="a[@code='or']">20</xsl:template>
    <xsl:template match="a[@code='changes']">21</xsl:template>
    <xsl:template match="a[@code='will']">22</xsl:template>
    <xsl:template match="a[@code='disappear']">24</xsl:template>
    <xsl:template match="a[@code='next']">25</xsl:template>
    <xsl:template match="a[@code='time']">26</xsl:template>
    <xsl:template match="a[@code='spec']">27</xsl:template>
    <xsl:template match="a[@code='rebuilt']">28</xsl:template>
    <xsl:template match="a[@code='problem']">29</xsl:template>
    <xsl:template match="a[@code='needs']">30</xsl:template>
    <xsl:template match="a[@code='be']">32</xsl:template>
    <xsl:template match="a[@code='fixed']">33</xsl:template>
    <xsl:template match="a[@code='at']">34</xsl:template>
    <xsl:template match="a[@code='source']">35</xsl:template>
    <xsl:template match="a[@code='generating']">36</xsl:template>
    <xsl:template match="a[@code='correct']">37</xsl:template>
    <xsl:template match="a[@code='from']">38</xsl:template>
    <xsl:template match="a[@code='examples']">39</xsl:template>
    <xsl:template match="a[@code='in']">40</xsl:template>
    <xsl:template match="a"><xsl:value-of select="string(@code)"/></xsl:template>
    
    <xsl:variable name="data">
        Unfortunately the attempt to fix this was done simply by modifying the generated tests. 
        This test set must not be manually modified, or the changes will simply disappear the next 
        time the spec is rebuilt. The problem needs to be fixed at source by generating correct 
        tests from the examples in the spec.
    </xsl:variable>
    
    <xsl:template name="main">
        <out>
            <xsl:for-each select="tokenize($data, '\s+')">
                <xsl:variable name="e" as="element(a)">
                    <a code="{replace(lower-case(.), '[^a-z]', '')}"/>
                </xsl:variable>
                <xsl:variable name="n" as="xs:string">
                    <xsl:apply-templates select="$e"/>
                </xsl:variable>
                <xsl:sequence select="$n"/>
            </xsl:for-each>
        </out>
    </xsl:template>
    
 
</xsl:stylesheet>
