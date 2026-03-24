<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:err="http://www.w3.org/2005/xqt-errors"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs err">
    
    <xsl:variable name="RUN" select="true()" static="yes"/>
    <xsl:strip-space elements="*"/>

    
    <!-- Test of xsl:source-document with xsl:value-of -->
    
    <xsl:template name="s-003" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="./BOOKLIST/BOOKS/ITEM/PRICE"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with xsl:value-of, filtered with a motionless predicate -->
    
    <xsl:template name="s-004" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="./BOOKLIST/BOOKS/ITEM[@CAT='P']/PRICE"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with xsl:value-of, filtered with a positional predicate -->
    
    <xsl:template name="s-005" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="./BOOKLIST/BOOKS/ITEM[position() lt 4]/PRICE"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with xsl:value-of, filtered with a positional predicate -->
    
    <xsl:template name="s-006" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="./BOOKLIST/BOOKS/*[position() lt 4]/PRICE"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with xsl:value-of, filtered with a positional predicate -->
    
    <xsl:template name="s-007" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="/BOOKLIST/BOOKS/*:ITEM[position() lt 4]/PRICE"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with xsl:value-of, filtered using data() to make it streamable -->
    
    <xsl:template name="s-008" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="./BOOKLIST/BOOKS/ITEM/PAGES/data()[. &lt; 1000][. &gt; 0]"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with xsl:value-of, filtered using text() to make it streamable -->
    
    <xsl:template name="s-009" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="BOOKLIST/BOOKS/ITEM/PAGES/text()[. &lt; 1000][. &gt; 0]"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with xsl:value-of, filtered using number() to make it streamable -->
    
    <xsl:template name="s-010" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="/BOOKLIST/BOOKS/ITEM/PAGES/number()[. &lt; 1000][. &gt; 0]"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with xsl:value-of, selecting nothing  -->
    
    <xsl:template name="s-011" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="ITEM/PAGES"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:value-of selecting both streamed nodes and literals -->
    
    <xsl:template name="s-012" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="./BOOKLIST/BOOKS/ITEM/PAGES/number(), 31, 32"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:value-of selecting both streamed nodes and literals while also filtering -->
    
    <xsl:template name="s-013" use-when="true() or $RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="tail(./BOOKLIST/BOOKS/ITEM/PAGES)/number(), 31, 32"/>
        </out>
      </xsl:source-document>
    </xsl:template>   
    
    <!-- Test of xsl:source-document with xsl:value-of of a computed value -->
    
    <xsl:template name="s-015" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="for $d in data(outermost(//DIMENSIONS))
                                return let $x := tokenize($d, '\s')!number() 
                                       return round($x[1]*$x[2]*$x[3], 3)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with xsl:value-of of a computed value -->
    
    <xsl:template name="s-016" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="account/transaction[position() lt 5]/(@value*2)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with xsl:value-of, filtered using copy-of() to make it streamable -->
    
    <xsl:template name="s-017" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="./BOOKLIST/BOOKS/ITEM/PAGES/copy-of()[. &lt; 1000][. &gt; 0]"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with xsl:value-of applied to an attribute -->
    
    <xsl:template name="s-018" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="account/transaction[position() lt 5]/@value"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document with xsl:value-of over attribute values, filtered -->
    
    <xsl:template name="s-019" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="account/transaction/@value[xs:decimal(.) lt -8.00]"/>
        </out>
      </xsl:source-document>
    </xsl:template> 
    
    <!-- Test of xsl:source-document with xsl:value-of over attribute values, computed -->
    
    <xsl:template name="s-020" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="account/transaction[position() mod 10 = 1]/abs(@value)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- xsl:value-of of a conditional value -->
    
    <xsl:template name="s-021" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/big-transactions.xml">
        <out>
          <xsl:value-of select="account/transaction/
                (if (xs:date(@date) lt xs:date('2020-01-01')) then +@value else (@value+1))"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- xsl:value-of with a loop used to compute each item -->
    
    <xsl:template name="s-022" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="BOOKLIST/BOOKS/ITEM/DIMENSIONS!xs:NMTOKENS(.)!xs:decimal(.)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Test of xsl:source-document calling xsl:value-of using //x/y -->
    <!-- Saxon makes this streamable by rewriting //X/Y as .//Y[parent::X] -->
    
    <xsl:template name="s-023" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="/BOOKLIST/BOOKS/ITEM/DIMENSIONS!xs:NMTOKENS(.)!xs:decimal(.)"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- within xsl:source-document, use xsl:value-of: argument crawling and consuming, nested nodes -->
  
    <xsl:template name="s-027" use-when="$RUN">
      <out>
        <xsl:source-document streamable="yes" href="../docs/nested-numbers.xml">
          <xsl:value-of select="descendant::n"/>
        </xsl:source-document>
      </out>
    </xsl:template>
    
    <!-- Streaming xsl:value-of: grounded operand, selects nothing -->
    
    <xsl:template name="s-040" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="data(/*/NOTHING)" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:value-of: striding operand, selects nothing -->
    
    <xsl:template name="s-041" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="/BOOKLIST/BOOKS/MAGAZINE" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:value-of: crawling operand, selects nothing -->
    
    <xsl:template name="s-042" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="/*/*/MAGAZINE" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:value-of: climbing operand, selects nothing -->
    
    <xsl:template name="s-043" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="//PRICE/../@nothing" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:value-of: separator is consuming -->
    
    <xsl:template name="s-044" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="1 to 10" separator="{substring(head(//AUTHOR), 1, 1)}"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:value-of: separator is consuming (and unused) -->
    
    <xsl:template name="s-045" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="(1 to 10)[current-date() lt xs:date('1999-11-16')]" separator="{substring(head(//AUTHOR), 1, 1)}"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:value-of: complex content -->
    
    <xsl:template name="s-046" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of>
            <xsl:for-each select="/BOOKLIST/BOOKS/ITEM">
              <xsl:text>[</xsl:text>
              <xsl:value-of select="PRICE"/>
              <xsl:text>]</xsl:text>
            </xsl:for-each>
          </xsl:value-of>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:value-of: descendant text nodes -->
    
    <xsl:template name="s-047" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="//PRICE/text()"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:value-of: striding element nodes -->
    
    <xsl:template name="s-070" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="/BOOKLIST/BOOKS/ITEM/PRICE" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:value-of: crawling element nodes -->
    
    <xsl:template name="s-071" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/books.xml">
        <out>
          <xsl:value-of select="//PRICE" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:value-of: crawling nested element nodes -->
    
    <xsl:template name="s-072" use-when="$RUN">
      <xsl:source-document streamable="yes" href="../docs/nested-numbers.xml">
        <out>
          <xsl:value-of select="//n" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:value-of: zero-length text nodes -->
    
    <xsl:template name="s-073" use-when="$RUN">
      <xsl:variable name="etn" as="text()">
        <xsl:value-of select="''"/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/nested-numbers.xml">
        <out>
          <xsl:value-of select="$etn, //n, $etn" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:value-of: adjacent text nodes -->
    
    <xsl:template name="s-074" use-when="$RUN">
      <xsl:variable name="gtn" as="text()">
        <xsl:value-of select="'~'"/>
      </xsl:variable>
      <xsl:source-document streamable="yes" href="../docs/nested-numbers.xml">
        <out>
          <xsl:value-of select="$gtn, //text(), $gtn" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:value-of: mix in atomic values -->
    
    <xsl:template name="s-075" use-when="$RUN">
      <xsl:variable name="av" as="xs:integer*" select="1 to 3"/>
      <xsl:source-document streamable="yes" href="../docs/nested-numbers.xml">
        <out>
          <xsl:value-of select="$av, //text(), $av" separator="|"/>
        </out>
      </xsl:source-document>
    </xsl:template>
    
    <!-- Streaming xsl:value-of: value-of document node -->
    
    <xsl:template name="s-076" use-when="$RUN">
      <xsl:variable name="av" as="xs:integer*" select="1 to 3"/>
      <xsl:source-document streamable="yes" href="../docs/nested-numbers.xml">
        <out>
          <xsl:value-of select="."/>
        </out>
      </xsl:source-document>
    </xsl:template>
    

                                                
    
</xsl:stylesheet>