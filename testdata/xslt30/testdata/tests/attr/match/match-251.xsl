<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="#all">
    
    <!-- Variant of match-249 using value comparisons-->
    
    <xsl:template match="a[@code eq 'unfortunately']">1</xsl:template>
    <xsl:template match="a[@code eq 'the']">2</xsl:template>
    <xsl:template match="a[@code eq 'attempt']">3</xsl:template>
    <xsl:template match="a[@code eq 'to']">4</xsl:template>
    <xsl:template match="a[@code eq 'fix']">5</xsl:template>
    <xsl:template match="a[@code eq 'this']">6</xsl:template>
    <xsl:template match="a[@code eq 'was']">7</xsl:template>
    <xsl:template match="a[@code eq 'done']">8</xsl:template>
    <xsl:template match="a[@code eq 'simply']">9</xsl:template>
    <xsl:template match="a[@code eq 'by']">10</xsl:template>
    <xsl:template match="a[@code eq 'modifying']">11</xsl:template>
    <xsl:template match="a[@code eq 'generated']">12</xsl:template>
    <xsl:template match="a[@code eq 'tests']">13</xsl:template>
    <xsl:template match="a[@code eq 'test']">14</xsl:template>
    <xsl:template match="a[@code eq 'set']">15</xsl:template>
    <xsl:template match="a[@code eq 'must']">16</xsl:template>
    <xsl:template match="a[@code eq 'not']">17</xsl:template>
    <xsl:template match="a[@code eq 'manually']">18</xsl:template>
    <xsl:template match="a[@code eq 'modified']">19</xsl:template>
    <xsl:template match="a[@code eq 'or']">20</xsl:template>
    <xsl:template match="a[@code eq 'changes']">21</xsl:template>
    <xsl:template match="a[@code eq 'will']">22</xsl:template>
    <xsl:template match="a[@code eq 'disappear']">24</xsl:template>
    <xsl:template match="a[@code eq 'next']">25</xsl:template>
    <xsl:template match="a[@code eq 'time']">26</xsl:template>
    <xsl:template match="a[@code eq 'spec']">27</xsl:template>
    <xsl:template match="a[@code eq 'rebuilt']">28</xsl:template>
    <xsl:template match="a[@code eq 'problem']">29</xsl:template>
    <xsl:template match="a[@code eq 'needs']">30</xsl:template>
    <xsl:template match="a[@code eq 'be']">32</xsl:template>
    <xsl:template match="a[@code eq 'fixed']">33</xsl:template>
    <xsl:template match="a[@code eq 'at']">34</xsl:template>
    <xsl:template match="a[@code eq 'source']">35</xsl:template>
    <xsl:template match="a[@code eq 'generating']">36</xsl:template>
    <xsl:template match="a[@code eq 'correct']">37</xsl:template>
    <xsl:template match="a[@code eq 'from']">38</xsl:template>
    <xsl:template match="a[@code eq 'examples']">39</xsl:template>
    <xsl:template match="a[@code eq 'in']">40</xsl:template>
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
