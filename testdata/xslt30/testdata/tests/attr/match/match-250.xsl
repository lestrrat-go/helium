<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="#all">
    
    <!-- Simulation of match patterns in DITA stylesheets -->
    
    <xsl:template match="*[contains(@class,' topic/unfortunately ')]">1</xsl:template>
    <xsl:template match="*[contains(@class,' topic/the ')]">2</xsl:template>
    <xsl:template match="*[contains(@class,' topic/attempt ')]">3</xsl:template>
    <xsl:template match="*[contains(@class,' topic/to ')]">4</xsl:template>
    <xsl:template match="*[contains(@class,' topic/fix ')]">5</xsl:template>
    <xsl:template match="*[contains(@class,' topic/this ')]">6</xsl:template>
    <xsl:template match="*[contains(@class,' topic/was ')]">7</xsl:template>
    <xsl:template match="*[contains(@class,' topic/done ')]">8</xsl:template>
    <xsl:template match="*[contains(@class,' topic/simply ')]">9</xsl:template>
    <xsl:template match="*[contains(@class,' topic/by ')]">10</xsl:template>
    <xsl:template match="*[contains(@class,' topic/modifying ')]">11</xsl:template>
    <xsl:template match="*[contains(@class,' topic/generated ')]">12</xsl:template>
    <xsl:template match="*[contains(@class,' topic/tests ')]">13</xsl:template>
    <xsl:template match="*[contains(@class,' topic/test ')]">14</xsl:template>
    <xsl:template match="*[contains(@class,' topic/set ')]">15</xsl:template>
    <xsl:template match="*[contains(@class,' topic/must ')]">16</xsl:template>
    <xsl:template match="*[contains(@class,' topic/not ')]">17</xsl:template>
    <xsl:template match="*[contains(@class,' topic/manually ')]">18</xsl:template>
    <xsl:template match="*[contains(@class,' topic/modified ')]">19</xsl:template>
    <xsl:template match="*[contains(@class,' topic/or ')]">20</xsl:template>
    <xsl:template match="*[contains(@class,' topic/changes ')]">21</xsl:template>
    <xsl:template match="*[contains(@class,' topic/will ')]">22</xsl:template>
    <xsl:template match="*[contains(@class,' topic/is ')]">23</xsl:template>
    <xsl:template match="*[contains(@class,' topic/disappear ')]">24</xsl:template>
    <xsl:template match="*[contains(@class,' topic/next ')]">25</xsl:template>
    <xsl:template match="*[contains(@class,' topic/time ')]">26</xsl:template>
    <xsl:template match="*[contains(@class,' topic/spec ')]">27</xsl:template>
    <xsl:template match="*[contains(@class,' topic/rebuilt ')]">28</xsl:template>
    <xsl:template match="*[contains(@class,' topic/problem ')]">29</xsl:template>
    <xsl:template match="*[contains(@class,' topic/needs ')]">30</xsl:template>
    <xsl:template match="*[contains(@class,' topic/be ')]">32</xsl:template>
    <xsl:template match="*[contains(@class,' topic/fixed ')]">33</xsl:template>
    <xsl:template match="*[contains(@class,' topic/at ')]">34</xsl:template>
    <xsl:template match="*[contains(@class,' topic/source ')]">35</xsl:template>
    <xsl:template match="*[contains(@class,' topic/generating ')]">36</xsl:template>
    <xsl:template match="*[contains(@class,' topic/correct ')]">37</xsl:template>
    <xsl:template match="*[contains(@class,' topic/from ')]">38</xsl:template>
    <xsl:template match="*[contains(@class,' topic/examples ')]">39</xsl:template>
    <xsl:template match="*[contains(@class,' topic/in ')]">40</xsl:template>
    <xsl:template match="*"><xsl:value-of select="string(@class)"/></xsl:template>
    
    <xsl:variable name="data">
        Unfortunately the attempt to fix this was done simply by modifying the generated tests. 
        This test set must not be manually modified, or the changes will simply disappear the next 
        time the spec is rebuilt. The problem needs to be fixed at source by generating correct 
        tests from the examples in the spec.
    </xsl:variable>
    
    <xsl:template name="main">
        <out>
            <xsl:for-each select="tokenize($data, '\s+')[. != '']">
                <xsl:variable name="e" as="element(a)">
                    <a class=" map/li topic/{replace(lower-case(.), '[^a-z]', '')} image/jpeg "/>
                </xsl:variable>
                <xsl:variable name="n" as="xs:string">
                    <xsl:apply-templates select="$e"/>
                </xsl:variable>
                <xsl:sequence select="$n"/>
            </xsl:for-each>
        </out>
    </xsl:template>
    
 
</xsl:stylesheet>
